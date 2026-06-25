import React from 'react';
import { useTranslation } from 'react-i18next';
import { Camera } from 'lucide-react';
import { Card, CardHeader, CardBody } from '../ui';
import { WorkOrder } from '../../services/workOrdersApi';
import { PhotoAnnotation } from './PhotoAnnotation';
import { BeforeAfterSlider } from './BeforeAfterSlider';

interface WODetailPhotosProps {
  workOrder: WorkOrder;
}

export const WODetailPhotos: React.FC<WODetailPhotosProps> = ({ workOrder }) => {
  const { t } = useTranslation();

  if (!workOrder.photos || workOrder.photos.length === 0) {
    return (
      <div className="text-center py-12">
        <Camera className="w-12 h-12 text-slate-300 dark:text-slate-600 mx-auto mb-3" />
        <p className="text-sm text-slate-500 dark:text-slate-400">
          {t('workOrder.noPhotos') || 'Фотографии не прикреплены'}
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Photo Annotation for first photo */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <Camera className="w-5 h-5 text-slate-600 dark:text-slate-400" />
          <span>{t('workOrder.photoAnnotation') || 'Аннотация на фото'}</span>
        </CardHeader>
        <CardBody>
          <PhotoAnnotation
            imageUrl={workOrder.photos[0]}
            readOnly={workOrder.status === 'completed' || workOrder.status === 'cancelled'}
          />
        </CardBody>
      </Card>

      {/* Before/After slider if 2+ photos */}
      {workOrder.photos.length >= 2 && (
        <Card>
          <CardHeader className="flex items-center gap-2">
            <Camera className="w-5 h-5 text-slate-600 dark:text-slate-400" />
            <span>{t('workOrder.beforeAfter') || 'Сравнение «До/После»'}</span>
          </CardHeader>
          <CardBody>
            <BeforeAfterSlider
              beforeImage={workOrder.photos[0]}
              afterImage={workOrder.photos[workOrder.photos.length - 1]}
              beforeLabel={t('workOrder.before') || 'До'}
              afterLabel={t('workOrder.after') || 'После'}
            />
          </CardBody>
        </Card>
      )}

      {/* All photos grid */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <Camera className="w-5 h-5 text-slate-600 dark:text-slate-400" />
          <span>
            {t('workOrder.allPhotos') || 'Все фотографии'} ({workOrder.photos.length})
          </span>
        </CardHeader>
        <CardBody>
          <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
            {workOrder.photos.map((photo, i) => (
              <a
                key={i}
                href={photo}
                target="_blank"
                rel="noopener noreferrer"
                className="block aspect-video rounded-lg overflow-hidden bg-slate-100 dark:bg-slate-800 hover:ring-2 hover:ring-blue-500 transition-all"
              >
                <img
                  src={photo}
                  alt={`${t('workOrder.photo') || 'Фото'} ${i + 1}`}
                  className="w-full h-full object-cover"
                />
              </a>
            ))}
          </div>
        </CardBody>
      </Card>
    </div>
  );
};
